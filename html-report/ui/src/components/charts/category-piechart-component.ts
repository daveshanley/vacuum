import { html, LitElement, TemplateResult } from 'lit';
import { BaseCSS } from '../../ts/base.css';
import { Chart } from 'chart.js';
import { property } from 'lit/decorators.js';

import {
  ArcElement,
  DoughnutController,
  Filler,
  Legend,
  Tooltip,
} from 'chart.js';

Chart.register(ArcElement, DoughnutController, Filler, Legend, Tooltip);

export class CategoryPiechartComponent extends LitElement {
  public chart: Chart;

  @property()
  public type: Chart.ChartType; // tslint:disable-line:no-reserved-keywords
  @property()
  public data: Chart.ChartData;
  @property()
  public options: Chart.ChartOptions;

  static get styles() {
    return [BaseCSS];
  }

  /**
   * Called when the dom first time updated. init chart.js data, add observe, and add resize listener
   */
  public firstUpdated(): void {
    // const data: Chart.ChartData = this.data || {};
    // const options: Chart.ChartOptions = this.options || {};
    if (!this.chart) {
      const ctx: CanvasRenderingContext2D = this.shadowRoot
        .querySelector('canvas')
        .getContext('2d');
      this.chart = new Chart(ctx, {
        type: 'doughnut',
        data: {
          labels: ['Red', 'Blue', 'Yellow', 'Green', 'Purple', 'Orange'],
          datasets: [
            {
              label: '# of Votes',
              data: [12, 19, 3, 5, 2, 3],
              backgroundColor: [
                'rgba(183, 86, 243, 0.2)',
                'rgba(174,56,179, 0.2)',
                'rgba(98,196,255, 0.2)',
                'rgba(104,255,220, 0.2)',
                'rgba(37,118,171, 0.2)',
                'rgba(255, 159, 64, 0.2)',
              ],
              borderColor: [
                'rgba(183, 86, 243, 1)',
                'rgb(174,56,179, 1)',
                'rgb(98,196,255, 1)',
                'rgb(104,255,220)',
                'rgb(37,118,171)',
                'rgba(255, 159, 64, 1)',
              ],
              borderWidth: 1,
            },
          ],
        },
      });
    }
    this.chart.data = this.observe(this.chart.data);
    // for (const prop of Object.keys(this.chart.data)) {
    //   //this.chart.data[prop] = this.observe(this.chart.data[prop]);
    //   console.log(this.chart.data[prop])
    // }
    // this.chart.data.datasets = this.chart.data.datasets.map((dataset: Chart.ChartDataSets) => {
    //   dataset.data = this.observe(dataset.data);
    //
    //   return this.observe(dataset);
    // });
    window.addEventListener('resize', () => {
      if (this.chart) {
        this.chart.resize();
      }
    });
  }

  /**
   * Use Proxy to watch object props change
   * @params obj
   */
  public observe<T extends object>(obj: T): T {
    const updateChart: () => void = this.updateChart;

    return new Proxy(obj, {
      set: (target: T, prop: string, val: unknown): boolean => {
        // @ts-ignore
        target[prop] = val;
        Promise.resolve().then(updateChart);

        return true;
      },
    });
  }

  /**
   * Use lit-html render Elements
   */
  public render(): void | TemplateResult {
    return html`
      <style>
        .chart-size {
          position: relative;
          width: 300px;
        }
        canvas {
          width: 200px;
          height: 200px;
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
