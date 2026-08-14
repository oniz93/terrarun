import 'package:equatable/equatable.dart';

class GPSPoint extends Equatable {
  final DateTime timestamp;
  final double lat;
  final double lng;
  final double? altitude;
  final double? speed;
  final double? horizontalAccuracy;
  final int? heartRate;

  const GPSPoint({
    required this.timestamp,
    required this.lat,
    required this.lng,
    this.altitude,
    this.speed,
    this.horizontalAccuracy,
    this.heartRate,
  });

  Map<String, dynamic> toJson() {
    return {
      'timestamp': timestamp.toIso8601String(),
      'lat': lat,
      'lng': lng,
      'altitude': altitude,
      'speed': speed,
      'horizontal_accuracy': horizontalAccuracy,
      'heart_rate': heartRate,
    };
  }

  @override
  List<Object?> get props => [timestamp, lat, lng, altitude, speed, horizontalAccuracy, heartRate];
}
