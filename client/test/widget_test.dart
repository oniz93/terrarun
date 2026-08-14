import 'package:flutter_test/flutter_test.dart';
import 'package:terrarun/core/api/api_client.dart';
import 'package:terrarun/core/storage/secure_storage.dart';
import 'package:terrarun/main.dart';

void main() {
  testWidgets('App renders login screen', (WidgetTester tester) async {
    final storage = SecureStorage();
    final api = ApiClient(storage);
    await tester.pumpWidget(
      TerrarunApp(storage: storage, api: api),
    );
    expect(find.text('Terrarun'), findsWidgets);
  });
}
