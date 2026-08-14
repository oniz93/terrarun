part of 'map_bloc.dart';

abstract class MapEvent extends Equatable {
  const MapEvent();

  @override
  List<Object?> get props => [];
}

class MapLoaded extends MapEvent {
  final double swLat;
  final double swLng;
  final double neLat;
  final double neLng;

  const MapLoaded({
    required this.swLat,
    required this.swLng,
    required this.neLat,
    required this.neLng,
  });

  @override
  List<Object?> get props => [swLat, swLng, neLat, neLng];
}

class MapMoved extends MapEvent {
  final double swLat;
  final double swLng;
  final double neLat;
  final double neLng;

  const MapMoved({
    required this.swLat,
    required this.swLng,
    required this.neLat,
    required this.neLng,
  });

  @override
  List<Object?> get props => [swLat, swLng, neLat, neLng];
}
