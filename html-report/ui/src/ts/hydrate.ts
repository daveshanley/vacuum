import type { CategoryStatsChartComponent } from '../components/charts/category-stats-chart-component';
import type { ReportStatistics } from '../model/chart-data';

declare global {
  interface Window {
    statistics: ReportStatistics;
  }
}

async function hydrate() {
  const chart =
    document.querySelector<CategoryStatsChartComponent>('category-piechart');
  if (!chart || !window.statistics) return;
  await customElements.whenDefined('category-piechart');
  await chart.updateComplete;
  chart.setChartData(window.statistics);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => void hydrate(), {
    once: true,
  });
} else {
  void hydrate();
}
