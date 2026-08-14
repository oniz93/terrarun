part of 'run_bloc.dart';

enum RunStatus { idle, active, ending, completed, error }

class RunState extends Equatable {
  final RunStatus status;
  final List<GPSPoint> points;
  final DateTime? startTime;
  final double distanceM;
  final String? runId;

  const RunState({
    this.status = RunStatus.idle,
    this.points = const [],
    this.startTime,
    this.distanceM = 0,
    this.runId,
  });

  RunState copyWith({
    RunStatus? status,
    List<GPSPoint>? points,
    DateTime? startTime,
    double? distanceM,
    String? runId,
  }) {
    return RunState(
      status: status ?? this.status,
      points: points ?? this.points,
      startTime: startTime ?? this.startTime,
      distanceM: distanceM ?? this.distanceM,
      runId: runId ?? this.runId,
    );
  }

  @override
  List<Object?> get props => [status, points, startTime, distanceM, runId];
}
