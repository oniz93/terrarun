import 'package:flutter_test/flutter_test.dart';
import 'package:h3_flutter_plus/h3_flutter_plus.dart';

void main() {
  test('H3 API check', () async {
    final h3 = const H3Factory().load();
    final unsigned = BigInt.parse('8b28de824216fff', radix: 16);
    try {
      final boundary = h3.cellToBoundary(unsigned); // BigInt (unsigned 64-bit)
      print('String works: ${boundary.length}');
    } catch (e) {
      print('String failed: $e');
    }
    
    try {
      // final boundary = h3.cellToBoundary(unsigned); // Try BigInt
      // print('BigInt works: ${boundary.length}');
    } catch (e) {
      // print('BigInt failed: $e');
    }
  });
}
