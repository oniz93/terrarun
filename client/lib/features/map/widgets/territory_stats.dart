import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/map_bloc.dart';

class TerritoryStatsOverlay extends StatelessWidget {
  const TerritoryStatsOverlay({super.key});

  @override
  Widget build(BuildContext context) {
    return Positioned(
      top: 50,
      left: 16,
      right: 16,
      child: BlocBuilder<MapBloc, MapState>(
        builder: (context, state) {
          int neonCount = 0;
          int umbraCount = 0;

          for (final hex in state.hexes.values) {
            if (hex.ownedBy == 'neon') {
              neonCount++;
            } else if (hex.ownedBy == 'umbra') {
              umbraCount++;
            }
          }

          final total = neonCount + umbraCount;
          final neonPct = total > 0 ? (neonCount / total * 100).toStringAsFixed(1) : '0.0';
          final umbraPct = total > 0 ? (umbraCount / total * 100).toStringAsFixed(1) : '0.0';

          return Card(
            color: Theme.of(context).cardColor.withOpacity(0.8),
            elevation: 4,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('NEON', style: TextStyle(color: Colors.red, fontWeight: FontWeight.bold)),
                      Text('$neonPct%', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
                    ],
                  ),
                  const Text('TERRITORY CONTROL', style: TextStyle(fontSize: 12, color: Colors.grey)),
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      const Text('UMBRA', style: TextStyle(color: Colors.blue, fontWeight: FontWeight.bold)),
                      Text('$umbraPct%', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
