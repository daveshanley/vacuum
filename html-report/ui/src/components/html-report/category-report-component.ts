import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';

export class CategoryReportComponent extends BaseComponent {
  static get styles() {
    const reportCss = css`
      /* something in here */
    `;

    return [reportCss];
  }

  render() {
    return html`<slot></slot>`;
  }
}
