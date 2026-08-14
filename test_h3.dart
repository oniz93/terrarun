import 'package:h3_flutter_plus/h3_flutter_plus.dart';

void main() {
  final h3 = const H3Factory().load();
  final indexStr = 608692844030885887.toRadixString(16);
  print(indexStr);
}
