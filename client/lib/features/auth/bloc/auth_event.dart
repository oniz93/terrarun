part of 'auth_bloc.dart';

abstract class AuthEvent extends Equatable {
  const AuthEvent();

  @override
  List<Object?> get props => [];
}

class AppStarted extends AuthEvent {
  const AppStarted();
}

class LoggedIn extends AuthEvent {
  final String accessToken;
  final String refreshToken;

  const LoggedIn({required this.accessToken, required this.refreshToken});

  @override
  List<Object?> get props => [accessToken, refreshToken];
}

class LoggedOut extends AuthEvent {
  const LoggedOut();
}
