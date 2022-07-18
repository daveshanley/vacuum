import {BaseComponent} from '../../ts/base-component';
import {html} from 'lit';
import {customElement, property} from 'lit/decorators.js';

@customElement('category-report')
export class CategoryReportComponent extends BaseComponent {
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
