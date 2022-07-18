import {html, LitElement} from 'lit';
import {
    BarController,
    BarElement,
    CategoryScale,
    Chart,
    ChartData,
    Filler,
    Legend,
    LinearScale,
    LogarithmicScale,
    Tooltip,
} from 'chart.js';
import {ReportStatistics} from '../../model/chart-data';

Chart.register(
  BarController,
  Filler,
  Legend,
  Tooltip,
  BarElement,
  CategoryScale,
  LinearScale,
  LogarithmicScale
);

export class CategoryStatsChartComponent extends LitElement {
  public chart: Chart;

  private _buildChartDataFromReport(data: ReportStatistics): ChartData {
    const chartData: ChartData = { labels: [], datasets: [] };

    const errorData: Array<number> = [];
    const warnData: Array<number> = [];
    const infoData: Array<number> = [];

    const labels = [];
    for (let x = 0; x < data.categoryStatistics.length; x++) {
      labels.push(data.categoryStatistics[x].categoryName);
      errorData.push(data.categoryStatistics[x].errors);
      warnData.push(data.categoryStatistics[x].warnings);
      infoData.push(data.categoryStatistics[x].info);
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
    ];

    chartData.labels = labels;
    return chartData;
  }

  public setChartData(data: ReportStatistics) {
    const chartData = this._buildChartDataFromReport(data);

    if (!this.chart) {
      const ctx: CanvasRenderingContext2D = this.shadowRoot
        .querySelector('canvas')
        .getContext('2d');
      this.chart = new Chart(ctx, {
        type: 'bar',
        options: {
          scales: {
            x: {
              stacked: true,
            },
            y: {
              stacked: true,
              type: 'logarithmic',
            },
          },
        },
        data: chartData,
      });
    }
  }

  /**
   * Called when the dom first time updated. init chart.js data, add observe, and add resize listener
   */
  public firstUpdated(): void {
    window.addEventListener('resize', () => {
      if (this.chart) {
        this.chart.resize();
      }
    });
  }

  public render() {
    return html`
      <style>
        .chart-size {
          position: relative;
          width: 450px;
        }
      </style>
      <div class="chart-size">
        <canvas></canvas>
      </div>
    `;
  }

  public updateChart = (): void => {
    if (this.chart) {
      this.chart.update();
    }
  };
}
