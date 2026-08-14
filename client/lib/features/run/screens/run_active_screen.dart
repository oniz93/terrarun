import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../bloc/run_bloc.dart';
import 'run_summary_screen.dart';
import '../../../core/api/api_client.dart';
import '../../../core/map/terrarun_map.dart';
import '../services/gps_service.dart';

class RunActiveScreen extends StatefulWidget {
  const RunActiveScreen({super.key});

  @override
  State<RunActiveScreen> createState() => _RunActiveScreenState();
}

class _RunActiveScreenState extends State<RunActiveScreen> {
  final MapController _mapController = MapController();
  Timer? _timer;
  bool _centered = false;

  @override
  void initState() {
    super.initState();
    _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    _mapController.dispose();
    super.dispose();
  }

  LatLng _runCenter(RunState state) {
    if (state.points.isNotEmpty) {
      return LatLng(state.points.first.lat, state.points.first.lng);
    }
    return const LatLng(49.2827, -123.1207);
  }

  List<Polyline> _buildRoutePolyline(RunState state) {
    if (state.points.length < 2) return const [];
    final points =
        state.points.map((p) => LatLng(p.lat, p.lng)).toList();
    return [
      Polyline(
        points: points,
        strokeWidth: 4.0,
        color: const Color(0xFF00E676),
      ),
    ];
  }

  List<Marker> _buildCurrentMarker(RunState state) {
    if (state.points.isEmpty) return const [];
    final p = state.points.last;
    return [
      Marker(
        point: LatLng(p.lat, p.lng),
        width: 18,
        height: 18,
        child: Container(
          decoration: BoxDecoration(
            color: Colors.greenAccent,
            shape: BoxShape.circle,
            border: Border.all(color: Colors.white, width: 2),
          ),
        ),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) => RunBloc(
        apiClient: context.read<ApiClient>(),
        gpsService: GPSService(), // Should ideally be provided higher up
      )..add(const StartRun()),
      child: BlocConsumer<RunBloc, RunState>(
        listener: (context, state) {
          if (state.status == RunStatus.completed) {
            Navigator.of(context).pushReplacement(
              MaterialPageRoute(
                builder: (_) => RunSummaryScreen(runState: state),
              ),
            );
            return;
          }

          // Center the camera on the first tracked point once.
          if (!_centered && state.points.isNotEmpty) {
            _centered = true;
            final p = state.points.first;
            _mapController.move(LatLng(p.lat, p.lng), 16.0);
          }
        },
        builder: (context, state) {
          return Scaffold(
            body: Stack(
              children: [
                TerrarunMap(
                  controller: _mapController,
                  initialCenter: _runCenter(state),
                  initialZoom: 16.0,
                  polylines: _buildRoutePolyline(state),
                  markers: _buildCurrentMarker(state),
                ),
                SafeArea(
                  child: Column(
                    children: [
                      _buildHeader(state),
                      const Spacer(),
                      _buildFooter(context, state),
                    ],
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _buildHeader(RunState state) {
    final duration = state.startTime != null
        ? DateTime.now().difference(state.startTime!)
        : Duration.zero;

    final pace = state.distanceM > 0
        ? (duration.inSeconds / (state.distanceM / 1000))
        : 0.0;

    final paceStr = pace > 0
        ? '${(pace ~/ 60)}:${(pace % 60).toInt().toString().padLeft(2, '0')}'
        : '-:--';

    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Colors.black.withValues(alpha: 0.8), Colors.transparent],
        ),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildStat('PACE', paceStr),
          _buildStat(
              'DISTANCE', '${(state.distanceM / 1000).toStringAsFixed(2)} km'),
          _buildStat('TIME', _formatDuration(duration)),
        ],
      ),
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

  Widget _buildFooter(BuildContext context, RunState state) {
    return Container(
      padding: const EdgeInsets.all(32),
      child: GestureDetector(
        onLongPress: () {
          context.read<RunBloc>().add(const EndRun());
        },
        child: Container(
          width: 80,
          height: 80,
          decoration: const BoxDecoration(
            shape: BoxShape.circle,
            color: Colors.red,
          ),
          child: const Center(
            child: Text(
              'HOLD\nTO STOP',
              textAlign: TextAlign.center,
              style: TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                  fontSize: 10),
            ),
          ),
        ),
      ),
    );
  }

  String _formatDuration(Duration d) {
    final hours = d.inHours;
    final minutes = d.inMinutes % 60;
    final seconds = d.inSeconds % 60;
    if (hours > 0) {
      return '$hours:${minutes.toString().padLeft(2, '0')}:${seconds.toString().padLeft(2, '0')}';
    }
    return '${minutes.toString().padLeft(2, '0')}:${seconds.toString().padLeft(2, '0')}';
  }
}
