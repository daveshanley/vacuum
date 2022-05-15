import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class CategoryReportComponent extends BaseComponent {
  static get styles() {
    const buttonCss = css`
      .category-buttons {
        margin-top: 20px;
        display: flex;
        flex-direction: row;
        flex-wrap: wrap;
        place-items: center stretch;
        justify-content: center;
      }
    `;

    return [BaseCSS, buttonCss];
  }

  render() {
    return html`
      <div>
        <slot></slot>
      </div>
    `;
  }
}
