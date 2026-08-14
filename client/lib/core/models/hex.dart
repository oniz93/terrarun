import 'package:equatable/equatable.dart';

class Hex extends Equatable {
  /// H3 cell index as an unsigned 64-bit value.
  ///
  /// Stored as a [BigInt] because H3 indexes exceed JavaScript's safe-integer
  /// range (2^53) and would otherwise lose precision on the web client. The
  /// backend sends this value as a JSON string.
  final BigInt h3Index;
  final double? centerLat;
  final double? centerLng;
  final String? ownedBy;
  final int hp;

  const Hex({
    required this.h3Index,
    this.centerLat,
    this.centerLng,
    this.ownedBy,
    required this.hp,
  });

  factory Hex.fromJson(Map<String, dynamic> json) {
    return Hex(
      // Accepts both a string (preferred, exact) and a number (legacy/fallback).
      h3Index: BigInt.parse(json['h3_index'].toString()),
      centerLat: json['center_lat'] != null ? (json['center_lat'] as num).toDouble() : null,
      centerLng: json['center_lng'] != null ? (json['center_lng'] as num).toDouble() : null,
      ownedBy: json['owned_by'] as String?,
      hp: (json['hp'] as num).toInt(),
    );
  }

  @override
  List<Object?> get props => [h3Index, centerLat, centerLng, ownedBy, hp];
}
