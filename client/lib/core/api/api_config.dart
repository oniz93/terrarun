class ApiConfig {
  // Defaults target a backend running on the same machine (e.g. `docker compose up`).
  // Override for a physical device with:
  //   flutter run --dart-define=API_BASE_URL=http://<mac-lan-ip>:8080
  static const String baseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );

  static const String wsUrl = String.fromEnvironment(
    'WS_URL',
    defaultValue: 'ws://localhost:8080/ws',
  );

  // Kept for potential Mapbox tile layers; the default map currently uses OSM.
  static const String mapboxToken = String.fromEnvironment(
    'MAPBOX_TOKEN',
    defaultValue: '',
  );
}
