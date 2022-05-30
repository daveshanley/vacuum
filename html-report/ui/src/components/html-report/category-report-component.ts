import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';

export class CategoryReportComponent extends BaseComponent {
  render() {
    return html`<slot></slot>`;
  }
}
