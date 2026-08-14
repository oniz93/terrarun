import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

/// Returns a fill colour for a territory hex based on its owner and HP.
///
/// [ownedBy] is expected to be 'neon', 'umbra', or null (neutral).
/// Opacity scales with HP so stronger hexes look more solid.
Color factionColor(String? ownedBy, int hp) {
  final clamped = hp.clamp(0, 10);
  final opacity = 0.15 + (clamped / 10.0) * 0.6;
  switch (ownedBy) {
    case 'neon':
      return Color.fromRGBO(255, 0, 0, opacity);
    case 'umbra':
      return Color.fromRGBO(0, 0, 255, opacity);
    default:
      return const Color.fromRGBO(0, 0, 0, 0.0);
  }
}

/// A thin, reusable wrapper around [FlutterMap].
///
/// This is the single place where the map engine is configured. It is used by
/// the map screen (with live territory polygons) and the run screens (with a
/// simple background map + optional route). The widget owns its [MapController]
/// unless one is passed in, so callers don't have to manage its lifecycle.
class TerrarunMap extends StatefulWidget {
  const TerrarunMap({
    super.key,
    this.initialCenter = const LatLng(49.2827, -123.1207),
    this.initialZoom = 13.0,
    this.controller,
    this.onReady,
    this.onPositionChanged,
    this.polygons = const [],
    this.polylines = const [],
    this.markers = const [],
    this.tileUrlTemplate = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
    this.showAttribution = true,
  });

  final LatLng initialCenter;
  final double initialZoom;

  /// Optional externally-owned controller (e.g. for camera control).
  final MapController? controller;

  final VoidCallback? onReady;

  /// Fired whenever the camera moves; receives the current visible bounds.
  final void Function(LatLngBounds bounds)? onPositionChanged;

  final List<Polygon> polygons;
  final List<Polyline> polylines;
  final List<Marker> markers;

  final String tileUrlTemplate;
  final bool showAttribution;

  @override
  State<TerrarunMap> createState() => _TerrarunMapState();
}

class _TerrarunMapState extends State<TerrarunMap> {
  late final MapController _controller;
  late final bool _ownsController;

  @override
  void initState() {
    super.initState();
    _ownsController = widget.controller == null;
    _controller = widget.controller ?? MapController();
  }

  @override
  void dispose() {
    if (_ownsController) {
      _controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FlutterMap(
      mapController: _controller,
      options: MapOptions(
        initialCenter: widget.initialCenter,
        initialZoom: widget.initialZoom,
        onMapReady: widget.onReady,
        onPositionChanged: (camera, hasGesture) {
          widget.onPositionChanged?.call(camera.visibleBounds);
        },
      ),
      children: [
        TileLayer(
          urlTemplate: widget.tileUrlTemplate,
          userAgentPackageName: 'com.terrarun.terrarun',
        ),
        if (widget.polygons.isNotEmpty) PolygonLayer(polygons: widget.polygons),
        if (widget.polylines.isNotEmpty) PolylineLayer(polylines: widget.polylines),
        if (widget.markers.isNotEmpty) MarkerLayer(markers: widget.markers),
        if (widget.showAttribution)
          const SimpleAttributionWidget(
            source: Text('OpenStreetMap contributors'),
          ),
      ],
    );
  }
}
