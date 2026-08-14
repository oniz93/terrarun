import 'dart:async';
import 'package:geolocator/geolocator.dart';
import '../../../core/models/gps_point.dart';

class GPSService {
  StreamSubscription<Position>? _positionStreamSubscription;
  final _controller = StreamController<GPSPoint>.broadcast();

  Stream<GPSPoint> get positionStream => _controller.stream;

  Future<bool> requestPermissions() async {
    bool serviceEnabled;
    LocationPermission permission;

    serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      return false;
    }

    permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
      if (permission == LocationPermission.denied) {
        return false;
      }
    }

    if (permission == LocationPermission.deniedForever) {
      return false;
    }

    return true;
  }

  void startTracking() {
    _positionStreamSubscription?.cancel();
    
    const locationSettings = LocationSettings(
      accuracy: LocationAccuracy.high,
      distanceFilter: 5,
    );

    _positionStreamSubscription = Geolocator.getPositionStream(
      locationSettings: locationSettings,
    ).listen((Position position) {
      final point = GPSPoint(
        timestamp: position.timestamp,
        lat: position.latitude,
        lng: position.longitude,
        altitude: position.altitude,
        speed: position.speed,
        horizontalAccuracy: position.accuracy,
      );
      _controller.add(point);
    });
  }

  void stopTracking() {
    _positionStreamSubscription?.cancel();
    _positionStreamSubscription = null;
  }

  void dispose() {
    stopTracking();
    _controller.close();
  }
}
