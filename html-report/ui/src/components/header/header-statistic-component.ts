import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';

@customElement('header-statistic')
export class HeaderStatisticComponent extends BaseComponent {
  static get styles() {
    const staticCss = css`
      span {
        display: block;
      }

      div {
        padding: 5px;
        min-width: 80px;
      }

      span.grade {
        font-size: 1.3rem;
        font-weight: bold;
      }

      span.label {
        font-size: var(--sl-font-size-xx-small);
        color: var(--font-color);
      }

      .error {
        background-color: var(--error-color-lowalpha);
        border: 1px solid var(--error-color);
        color: var(--error-color);
      }

      .warn-400 {
        background-color: var(--warn-400-lowalpha);
        border: 1px solid var(--warn-400);
        color: var(--warn-400);
      }

      .warn-300 {
        background-color: var(--warn-300-lowalpha);
        border: 1px solid var(--warn-300);
        color: var(--warn-300);
      }

      .warn-200 {
        background-color: var(--warn-200-lowalpha);
        border: 1px solid var(--warn-200);
        color: var(--warn-200);
      }

      .warn {
        background-color: var(--warn-color-lowalpha);
        border: 1px solid var(--warn-color);
        color: var(--warn-color);
      }

      .ok-400 {
        background-color: var(--ok-400-lowalpha);
        border: 1px solid var(--ok-400);
        color: var(--ok-400);
      }

      .ok-300 {
        background-color: var(--ok-300-lowalpha);
        border: 1px solid var(--ok-300);
        color: var(--ok-300);
      }

      .ok-200 {
        background-color: var(--ok-200-lowalpha);
        border: 1px solid var(--ok-200);
        color: var(--ok-200);
      }

      .ok {
        background-color: var(--ok-color-lowalpha);
        border: 1px solid var(--ok-color);
        color: var(--ok-color);
      }

      .warning-count {
        background: none;
        color: var(--primary-color);
      }

      .error-count {
        background: none;
        color: var(--primary-color);
      }

      .info-count {
        background: none;
        color: var(--primary-color);
      }

      @media only screen and (max-width: 600px) {
        div {
          padding: 5px;
          min-width: 60px;
        }
      }
    `;
    return [staticCss];
  }

  @property({ type: Number })
  value: number;

  @property()
  preset: string;

  @property()
  percentage: boolean;

  @property()
  label: string;

  render() {
    return html`
      <div class=${this.colorForScore()}>
        <span class="grade"
          >${this.value.toLocaleString()}${this.percentage ? '%' : ''}</span
        >
        <span class="label"> ${this.label} </span>
      </div>
    `;
  }

  colorForScore(): string {
    if (this.preset) {
      return this.preset;
    }

    switch (true) {
      case this.value <= 10:
        return 'error';

      case this.value > 10 && this.value < 20:
        return 'warn-400';

      case this.value >= 20 && this.value < 30:
        return 'warn-300';

      case this.value >= 30 && this.value < 40:
        return 'warn-200';

      case this.value >= 40 && this.value < 50:
        return 'warn';

      case this.value >= 50 && this.value < 65:
        return 'ok-400';

      case this.value >= 65 && this.value < 75:
        return 'ok-300';

      case this.value >= 75 && this.value < 95:
        return 'ok-200';

      case this.value >= 95:
        return 'ok';

      default:
        return 'ok';
    }
  }
}
