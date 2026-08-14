part of 'map_bloc.dart';

enum MapStatus { initial, loading, success, failure }

class MapState extends Equatable {
  final MapStatus status;
  final Map<BigInt, Hex> hexes;

  const MapState({
    this.status = MapStatus.initial,
    this.hexes = const {},
  });

  factory MapState.initial() =>
      const MapState(status: MapStatus.initial, hexes: {});

  MapState copyWith({
    MapStatus? status,
    Map<BigInt, Hex>? hexes,
  }) {
    return MapState(
      status: status ?? this.status,
      hexes: hexes ?? this.hexes,
    );
  }

  @override
  List<Object?> get props => [status, hexes];
}
