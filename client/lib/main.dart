import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'core/api/api_client.dart';
import 'core/storage/secure_storage.dart';
import 'features/auth/bloc/auth_bloc.dart';
import 'features/auth/screens/home_screen.dart';
import 'features/auth/screens/login_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final storage = SecureStorage();
  final api = ApiClient(storage);

  runApp(TerrarunApp(storage: storage, api: api));
}

class TerrarunApp extends StatelessWidget {
  final SecureStorage storage;
  final ApiClient api;

  TerrarunApp({
    super.key,
    required this.storage,
    required this.api,
  });

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider<AuthBloc>(
          create: (_) => AuthBloc(storage)..add(const AppStarted()),
        ),
        RepositoryProvider<ApiClient>.value(value: api),
      ],
      child: MaterialApp(
        title: 'Terrarun',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorSchemeSeed: const Color(0xFF00E676),
          brightness: Brightness.dark,
          useMaterial3: true,
        ),
        home: BlocBuilder<AuthBloc, AuthState>(
          builder: (context, state) {
            if (state is AuthAuthenticated) {
              return const HomeScreen();
            }
            return const LoginScreen();
          },
        ),
      ),
    );
  }
}
