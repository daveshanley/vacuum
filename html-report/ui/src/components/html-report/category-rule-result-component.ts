import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class CategoryRuleResultComponent extends BaseComponent {
  static get styles() {
    return [BaseCSS];
  }

  render() {
    return html`
      <div>
        <slot></slot>
      </div>`
  }
}