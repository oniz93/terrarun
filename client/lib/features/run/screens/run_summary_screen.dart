import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../bloc/run_bloc.dart';
import '../../../core/map/terrarun_map.dart';

class RunSummaryScreen extends StatelessWidget {
  final RunState runState;

  const RunSummaryScreen({super.key, required this.runState});

  LatLng _center() {
    if (runState.points.isNotEmpty) {
      return LatLng(runState.points.first.lat, runState.points.first.lng);
    }
    return const LatLng(49.2827, -123.1207);
  }

  List<Polyline> _buildRoutePolyline() {
    if (runState.points.length < 2) return const [];
    final points =
        runState.points.map((p) => LatLng(p.lat, p.lng)).toList();
    return [
      Polyline(
        points: points,
        strokeWidth: 4.0,
        color: const Color(0xFF00E676),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Run Summary'),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ),
      body: SingleChildScrollView(
        child: Column(
          children: [
            _buildMap(context),
            Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                children: [
                  _buildMainStats(context),
                  const Divider(height: 48),
                  _buildTerritorySummary(context),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMap(BuildContext context) {
    return SizedBox(
      height: 300,
      child: TerrarunMap(
        initialCenter: _center(),
        initialZoom: 15.0,
        polylines: _buildRoutePolyline(),
      ),
    );
  }

  Widget _buildMainStats(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceAround,
      children: [
        _buildStat(
            'DISTANCE', '${(runState.distanceM / 1000).toStringAsFixed(2)} km'),
        _buildStat('POINTS', '0'), // TODO: Get from backend
      ],
    );
  }

  Widget _buildStat(String label, String value) {
    return Column(
      children: [
        Text(label,
            style: const TextStyle(
                color: Colors.grey,
                fontSize: 12,
                fontWeight: FontWeight.bold)),
        Text(value,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.bold)),
      ],
    );
  }

  Widget _buildTerritorySummary(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: Colors.grey[900],
        borderRadius: BorderRadius.circular(16),
      ),
      child: const Column(
        children: [
          Text('TERRITORY CAPTURED',
              style: TextStyle(
                  color: Colors.grey, fontWeight: FontWeight.bold)),
          SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.hexagon, color: Colors.red),
              SizedBox(width: 8),
              Text('0 HEXES',
                  style: TextStyle(
                      fontSize: 24, fontWeight: FontWeight.bold)),
            ],
          ),
        ],
      ),
    );
  }
}
