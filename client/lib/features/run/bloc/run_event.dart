part of 'run_bloc.dart';

abstract class RunEvent extends Equatable {
  const RunEvent();

  @override
  List<Object?> get props => [];
}

class StartRun extends RunEvent {
  const StartRun();
}

class UpdatePosition extends RunEvent {
  final GPSPoint point;
  const UpdatePosition(this.point);

  @override
  List<Object?> get props => [point];
}

class EndRun extends RunEvent {
  const EndRun();
}
