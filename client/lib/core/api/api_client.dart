import 'package:dio/dio.dart';
import '../storage/secure_storage.dart';
import 'api_config.dart';

class ApiClient {
  late final Dio _dio;
  final SecureStorage _storage;

  ApiClient(this._storage) {
    _dio = Dio(BaseOptions(
      baseUrl: ApiConfig.baseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 30),
      headers: {'Content-Type': 'application/json'},
    ));

    _dio.interceptors.add(QueuedInterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _storage.getAccessToken();
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        if (error.response?.statusCode == 401) {
          final refreshToken = await _storage.getRefreshToken();
          if (refreshToken != null) {
            try {
              // Using a separate Dio instance for refresh to avoid interceptor loops
              final refreshDio = Dio();
              final resp = await refreshDio.post(
                '${ApiConfig.baseUrl}/auth/refresh',
                data: {'refresh_token': refreshToken},
              );
              
              final newAccess = resp.data['access_token'] as String;
              final newRefresh = resp.data['refresh_token'] as String;
              
              await _storage.saveTokens(
                accessToken: newAccess,
                refreshToken: newRefresh,
              );
              
              // Update the original request's header
              error.requestOptions.headers['Authorization'] = 'Bearer $newAccess';
              
              // Retry the request
              final retryResp = await _dio.fetch(error.requestOptions);
              return handler.resolve(retryResp);
            } catch (e) {
              await _storage.clear();
              return handler.next(error);
            }
          }
        }
        handler.next(error);
      },
    ));
  }

  Future<Response> get(String path, {Map<String, dynamic>? queryParams}) =>
      _dio.get(path, queryParameters: queryParams);

  Future<Response> post(String path, {dynamic data}) =>
      _dio.post(path, data: data);

  Future<Response> put(String path, {dynamic data}) =>
      _dio.put(path, data: data);

  Future<Response> delete(String path) => _dio.delete(path);
}
