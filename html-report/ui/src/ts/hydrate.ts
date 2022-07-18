import {CategoryStatsChartComponent} from '../components/charts/category-stats-chart-component';

declare global {
  interface Window {
    statistics: never;
  }
}

// give the browser a few ms to breathe after setting up every component
setTimeout(hydrate, 200);
function hydrate() {
  // when we need to hydrate post render.
  // chart needs data
  const catChart = document.querySelector(
    'category-piechart'
  ) as CategoryStatsChartComponent;
  if (catChart) {
    catChart.setChartData(window.statistics);
  } else {
    //alert('Something went really wrong.');
  }
}
