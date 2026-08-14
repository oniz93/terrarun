import 'package:flutter_test/flutter_test.dart';
import 'package:h3_flutter_plus/h3_flutter_plus.dart';

void main() {
  test('H3 LatLng check', () async {
    final h3 = const H3Factory().load();
    final unsigned = BigInt.parse('8b28de824216fff', radix: 16);
    final boundary = h3.cellToBoundary(unsigned);
    for (var ll in boundary) {
      print('Lat: ${ll.lat}, Lng: ${ll.lng}');
    }
  });
}
