import 'package:flutter_test/flutter_test.dart';

void main() {
  test('H3 int64 conversion', () {
    int h3Index = 608692844030885887;
    String hexStr = h3Index.toRadixString(16);
    print(hexStr);
    expect(hexStr, '87283082a03ffff'); // Or similar depending on correct hex
  });
}
