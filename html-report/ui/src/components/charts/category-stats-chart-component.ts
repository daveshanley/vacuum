import { html, LitElement } from 'lit';
import {
  BarController,
  BarElement,
  CategoryScale,
  Chart,
  ChartData,
  Filler,
  Legend,
  LinearScale,
  Tooltip,
} from 'chart.js';
import { ReportStatistics } from '../../model/chart-data';

Chart.register(
  BarController,
  Filler,
  Legend,
  Tooltip,
  BarElement,
  CategoryScale,
  LinearScale,
);

export class CategoryStatsChartComponent extends LitElement {
  public chart: Chart;

  private _buildChartDataFromReport(data: ReportStatistics): ChartData {
    const chartData: ChartData = { labels: [], datasets: [] };

    const errorData: Array<number> = [];
    const warnData: Array<number> = [];
    const infoData: Array<number> = [];
    const hintData: Array<number> = [];

    const labels = [];
    const stats = data.categoryStatistics ?? [];
    for (const category of stats) {
      labels.push(category.categoryName);
      errorData.push(category.errors);
      warnData.push(category.warnings);
      infoData.push(category.info);
      hintData.push(category.hints);
    }

    chartData.datasets = [
      {
        label: 'Errors',
        data: errorData,
        backgroundColor: ['rgba(255,0,0,0.2)'],
        borderColor: ['rgb(255,0,0)'],
        borderWidth: 1,
      },
      {
        label: 'Warnings',
        data: warnData,
        backgroundColor: ['rgba(255,174,0,0.2)'],
        borderColor: ['rgb(255,145,0)'],
        borderWidth: 1,
      },
      {
        label: 'Information',
        data: infoData,
        backgroundColor: ['rgba(0,158,255,0.2)'],
        borderColor: ['rgb(0,162,255)'],
        borderWidth: 1,
      },
      {
        label: 'Hints',
        data: hintData,
        backgroundColor: ['rgba(26,255,0,0.2)'],
        borderColor: ['rgb(26,255,0)'],
        borderWidth: 1,
      },
    ];

    chartData.labels = labels;
    return chartData;
  }

  /** Render category counts after the component has completed its first update. */
  public setChartData(data: ReportStatistics) {
    const chartData = this._buildChartDataFromReport(data);

    if (!this.chart) {
      const ctx: CanvasRenderingContext2D = this.shadowRoot
        .querySelector('canvas')
        .getContext('2d');
      this.chart = new Chart(ctx, {
        type: 'bar',
        options: {
          maintainAspectRatio: false,
          plugins: { legend: { labels: { color: '#e8e9ed' } } },
          scales: {
            x: {
              stacked: true,
              ticks: { color: '#a3abba' },
            },
            y: {
              stacked: true,
              beginAtZero: true,
              ticks: { color: '#a3abba', precision: 0 },
              grid: { color: '#3d3d3d' },
            },
          },
        },
        data: chartData,
      });
    } else {
      this.chart.data = chartData;
      this.chart.update();
    }
  }

  /** Release canvas observers and event handlers when the component is removed. */
  public disconnectedCallback(): void {
    this.chart?.destroy();
    this.chart = undefined;
    super.disconnectedCallback();
  }

  public render() {
    return html`
      <style>
        .chart-size {
          position: relative;
          width: 100%;
          height: 300px;
        }
      </style>
      <div class="chart-size">
        <canvas
          role="img"
          aria-label="Violations by category and severity"
        ></canvas>
      </div>
    `;
  }

  public updateChart = (): void => {
    if (this.chart) {
      this.chart.update();
    }
  };
}
