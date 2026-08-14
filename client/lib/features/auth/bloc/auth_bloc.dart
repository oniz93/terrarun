import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../../../core/storage/secure_storage.dart';

part 'auth_event.dart';
part 'auth_state.dart';

class AuthBloc extends Bloc<AuthEvent, AuthState> {
  final SecureStorage _storage;

  AuthBloc(this._storage) : super(const AuthInitial()) {
    on<AppStarted>(_onAppStarted);
    on<LoggedIn>(_onLoggedIn);
    on<LoggedOut>(_onLoggedOut);
  }

  Future<void> _onAppStarted(AppStarted event, Emitter<AuthState> emit) async {
    final token = await _storage.getAccessToken();
    if (token != null) {
      emit(const AuthAuthenticated());
    } else {
      emit(const AuthUnauthenticated());
    }
  }

  Future<void> _onLoggedIn(LoggedIn event, Emitter<AuthState> emit) async {
    await _storage.saveTokens(
      accessToken: event.accessToken,
      refreshToken: event.refreshToken,
    );
    emit(const AuthAuthenticated());
  }

  Future<void> _onLoggedOut(LoggedOut event, Emitter<AuthState> emit) async {
    await _storage.clear();
    emit(const AuthUnauthenticated());
  }
}
