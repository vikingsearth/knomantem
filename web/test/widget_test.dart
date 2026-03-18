import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:knomantem/app.dart';

void main() {
  testWidgets('App renders without crashing', (WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: KnomantemApp(),
      ),
    );
    // Allow navigation/auth to resolve
    await tester.pump(const Duration(milliseconds: 100));
    // Should show login screen (unauthenticated)
    expect(find.byType(MaterialApp), findsOneWidget);
  });

  testWidgets('Login screen has email and password fields',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: KnomantemApp(),
      ),
    );
    await tester.pump(const Duration(milliseconds: 300));

    // Should have navigated to login
    expect(find.byType(TextFormField), findsWidgets);
  });
}
