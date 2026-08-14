import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:equatable/equatable.dart';
import 'dart:async';
import 'package:geolocator/geolocator.dart';
import '../../../core/api/api_client.dart';
import '../../../core/models/gps_point.dart';
import '../services/gps_service.dart';
import 'package:latlong2/latlong.dart';

part 'run_event.dart';
part 'run_state.dart';

class RunBloc extends Bloc<RunEvent, RunState> {
  final ApiClient apiClient;
  final GPSService gpsService;
  StreamSubscription? _gpsSubscription;

  RunBloc({
    required this.apiClient,
    required this.gpsService,
  }) : super(const RunState()) {
    on<StartRun>(_onStartRun);
    on<UpdatePosition>(_onUpdatePosition);
    on<EndRun>(_onEndRun);
  }

  Future<void> _onStartRun(StartRun event, Emitter<RunState> emit) async {
    final hasPermission = await gpsService.requestPermissions();
    if (!hasPermission) {
      emit(state.copyWith(status: RunStatus.error));
      return;
    }

    try {
      // 1. Get initial position to start the run
      final position = await Geolocator.getCurrentPosition();
      
      // 2. Notify backend
      final response = await apiClient.post('/runs/start', data: {
        'lat': position.latitude,
        'lng': position.longitude,
        'social_run': false, // TODO: support social runs
        'social_participants': [],
      });

      final runId = response.data['id'] as String;
      
      emit(state.copyWith(
        status: RunStatus.active, 
        startTime: DateTime.now(), 
        points: [],
        runId: runId,
      ));
      
      gpsService.startTracking();
      _gpsSubscription = gpsService.positionStream.listen((GPSPoint point) {
        add(UpdatePosition(point));
      });
    } catch (e) {
      emit(state.copyWith(status: RunStatus.error));
    }
  }

  void _onUpdatePosition(UpdatePosition event, Emitter<RunState> emit) {
    if (state.status != RunStatus.active) return;

    final newPoints = List<GPSPoint>.from(state.points)..add(event.point);
    
    // Calculate distance
    double totalDistance = state.distanceM;
    if (newPoints.length > 1) {
      final p1 = newPoints[newPoints.length - 2];
      final p2 = newPoints.last;
      final distance = const Distance().as(
        LengthUnit.Meter,
        LatLng(p1.lat, p1.lng),
        LatLng(p2.lat, p2.lng),
      );
      totalDistance += distance;
    }

    emit(state.copyWith(
      points: newPoints,
      distanceM: totalDistance,
    ));
  }

  Future<void> _onEndRun(EndRun event, Emitter<RunState> emit) async {
    if (state.status != RunStatus.active || state.runId == null) return;

    emit(state.copyWith(status: RunStatus.ending));
    
    gpsService.stopTracking();
    _gpsSubscription?.cancel();

    try {
      final durationS = DateTime.now().difference(state.startTime!).inSeconds;
      
      await apiClient.post('/runs/${state.runId}/end', data: {
        'gps_points': state.points.map((p) => p.toJson()).toList(),
        'distance_m': state.distanceM,
        'duration_s': durationS,
      });
      
      // TODO: Parse summary response and update state
      
      emit(state.copyWith(status: RunStatus.completed));
    } catch (e) {
      emit(state.copyWith(status: RunStatus.error));
    }
  }

  @override
  Future<void> close() {
    _gpsSubscription?.cancel();
    return super.close();
  }
}
