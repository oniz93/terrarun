import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:h3_flutter_plus/h3_flutter_plus.dart' as h3p;
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../bloc/map_bloc.dart';
import '../widgets/run_controls.dart';
import '../widgets/territory_stats.dart';
import '../../run/screens/run_active_screen.dart';
import '../../../core/api/api_client.dart';
import '../../../core/map/terrarun_map.dart';
import '../../../core/models/hex.dart';

class MapScreen extends StatefulWidget {
  const MapScreen({super.key});

  @override
  State<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends State<MapScreen> {
  final MapController _mapController = MapController();
  late final h3p.H3 h3;
  bool _h3Loaded = false;
  bool _mapReady = false;
  geo.Position? _currentPosition;

  @override
  void initState() {
    super.initState();
    _initH3();
    _determinePosition();
  }

  @override
  void dispose() {
    _mapController.dispose();
    super.dispose();
  }

  Future<void> _determinePosition() async {
    bool serviceEnabled;
    geo.LocationPermission permission;

    serviceEnabled = await geo.Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      print('Location services are disabled.');
      return;
    }

    permission = await geo.Geolocator.checkPermission();
    if (permission == geo.LocationPermission.denied) {
      permission = await geo.Geolocator.requestPermission();
      if (permission == geo.LocationPermission.denied) {
        print('Location permissions are denied');
        return;
      }
    }

    if (permission == geo.LocationPermission.deniedForever) {
      print('Location permissions are permanently denied');
      return;
    }

    final position = await geo.Geolocator.getCurrentPosition();
    if (mounted) {
      setState(() {
        _currentPosition = position;
      });

      if (_mapReady) {
        _mapController.move(
          LatLng(position.latitude, position.longitude),
          14.0,
        );
      }
    }
  }

  Future<void> _initH3() async {
    try {
      h3 = const h3p.H3Factory().load();
      if (mounted) {
        setState(() => _h3Loaded = true);
      }
      print('H3 library loaded successfully');
    } catch (e) {
      print('Error loading H3 library: $e');
    }
  }

  void _onMapReady() {
    _mapReady = true;
    if (_currentPosition != null) {
      _mapController.move(
        LatLng(_currentPosition!.latitude, _currentPosition!.longitude),
        14.0,
      );
    }
    _dispatchViewportEvent(true);
  }

  void _onPositionChanged(LatLngBounds bounds) {
    if (!_mapReady) return;
    _dispatchViewportEvent(false);
  }

  void _dispatchViewportEvent(bool isInitial) {
    if (!_mapReady || !mounted) return;

    final bounds = _mapController.camera.visibleBounds;
    final sw = bounds.southWest;
    final ne = bounds.northEast;

    final bloc = context.read<MapBloc>();
    if (isInitial) {
      bloc.add(MapLoaded(
        swLat: sw.latitude,
        swLng: sw.longitude,
        neLat: ne.latitude,
        neLng: ne.longitude,
      ));
    } else {
      bloc.add(MapMoved(
        swLat: sw.latitude,
        swLng: sw.longitude,
        neLat: ne.latitude,
        neLng: ne.longitude,
      ));
    }
  }

  List<Polygon> _buildHexPolygons(Map<BigInt, Hex> hexes) {
    final polygons = <Polygon>[];
    for (final hex in hexes.values) {
      try {
        final boundary = h3.cellToBoundary(hex.h3Index);
        final points =
            boundary.map((latLng) => LatLng(latLng.lat, latLng.lng)).toList();
        if (points.isEmpty) continue;

        polygons.add(Polygon(
          points: points,
          color: factionColor(hex.ownedBy, hex.hp),
          borderColor: Colors.white.withValues(alpha: 0.10),
          borderStrokeWidth: 1.0,
        ));
      } catch (e) {
        print('Error decoding hex ${hex.h3Index}: $e');
      }
    }
    return polygons;
  }

  List<Marker> _buildMarkers() {
    final pos = _currentPosition;
    if (pos == null) return const [];

    return [
      Marker(
        point: LatLng(pos.latitude, pos.longitude),
        width: 20,
        height: 20,
        child: Container(
          decoration: BoxDecoration(
            color: Colors.white,
            shape: BoxShape.circle,
            border: Border.all(color: Colors.blueAccent, width: 3),
            boxShadow: const [BoxShadow(color: Colors.black26, blurRadius: 4)],
          ),
        ),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    if (!_h3Loaded) {
      return const Center(child: CircularProgressIndicator());
    }

    return BlocProvider(
      create: (context) => MapBloc(context.read<ApiClient>()),
      child: Builder(
        builder: (context) {
          return Scaffold(
            body: Stack(
              children: [
                BlocBuilder<MapBloc, MapState>(
                  builder: (context, state) {
                    return TerrarunMap(
                      controller: _mapController,
                      initialCenter: _currentPosition != null
                          ? LatLng(_currentPosition!.latitude,
                              _currentPosition!.longitude)
                          : const LatLng(49.2827, -123.1207),
                      initialZoom: 14.0,
                      onReady: _onMapReady,
                      onPositionChanged: _onPositionChanged,
                      polygons: _buildHexPolygons(state.hexes),
                      markers: _buildMarkers(),
                    );
                  },
                ),
                const TerritoryStatsOverlay(),
                RunControls(
                  onStartRun: () {
                    Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => const RunActiveScreen(),
                      ),
                    );
                  },
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
