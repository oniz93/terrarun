import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:equatable/equatable.dart';
import 'package:terrarun/core/models/hex.dart';
import 'package:terrarun/core/api/api_client.dart';
import 'package:rxdart/rxdart.dart';

part 'map_event.dart';
part 'map_state.dart';

class MapBloc extends Bloc<MapEvent, MapState> {
  final ApiClient apiClient;

  MapBloc(this.apiClient) : super(MapState.initial()) {
    on<MapLoaded>(_onMapLoaded);
    on<MapMoved>(
      _onMapMoved,
      transformer: (events, mapper) => events
          .debounceTime(const Duration(milliseconds: 500))
          .switchMap(mapper),
    );
  }

  Future<void> _onMapLoaded(MapLoaded event, Emitter<MapState> emit) async {
    emit(state.copyWith(status: MapStatus.loading));
    await _fetchTerritory(event.swLat, event.swLng, event.neLat, event.neLng, emit);
  }

  Future<void> _onMapMoved(MapMoved event, Emitter<MapState> emit) async {
    await _fetchTerritory(event.swLat, event.swLng, event.neLat, event.neLng, emit);
  }

  Future<void> _fetchTerritory(
    double swLat,
    double swLng,
    double neLat,
    double neLng,
    Emitter<MapState> emit,
  ) async {
    try {
      final response = await apiClient.get('/territory', queryParams: {
        'sw_lat': swLat,
        'sw_lng': swLng,
        'ne_lat': neLat,
        'ne_lng': neLng,
      });

      final List<dynamic> hexesData = response.data['hexes'] ?? [];
      final Map<BigInt, Hex> newHexes = {};
      
      print('Fetched ${hexesData.length} hexes from API');
      if (hexesData.isNotEmpty) {
        print('Sample hex: ${hexesData[0]}');
      }

      for (var json in hexesData) {
        final hex = Hex.fromJson(json);
        newHexes[hex.h3Index] = hex;
      }

      emit(state.copyWith(
        status: MapStatus.success,
        hexes: newHexes,
      ));
    } catch (e) {
      print('Error fetching territory: $e');
      emit(state.copyWith(status: MapStatus.failure));
    }
  }
}
