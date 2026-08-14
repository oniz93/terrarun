import 'package:flutter/material.dart';

class RunControls extends StatelessWidget {
  final VoidCallback onStartRun;

  const RunControls({super.key, required this.onStartRun});

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.bottomCenter,
      child: Padding(
        padding: const EdgeInsets.only(bottom: 32.0),
        child: GestureDetector(
          onVerticalDragEnd: (details) {
            if (details.primaryVelocity != null && details.primaryVelocity! < -10) {
              // Swipe up detected
              onStartRun();
            }
          },
          child: Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: Theme.of(context).colorScheme.primary,
              boxShadow: [
                BoxShadow(
                  color: Theme.of(context).colorScheme.primary.withOpacity(0.4),
                  blurRadius: 20,
                  spreadRadius: 5,
                )
              ],
            ),
            child: const Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.keyboard_double_arrow_up, color: Colors.white, size: 24),
                Text('START', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
