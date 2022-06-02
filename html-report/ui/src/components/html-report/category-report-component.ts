import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { property } from 'lit/decorators.js';

export class CategoryReportComponent extends BaseComponent {
  static get styles() {
    const reportCss = css`
      /* something in here */
    `;

    return [reportCss];
  }

  @property()
  id: string;

  get results() {
    return this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });
  }

  render() {
    return html`<slot></slot>`;
  }
}
