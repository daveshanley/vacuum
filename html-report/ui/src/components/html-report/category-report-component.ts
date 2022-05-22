import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class CategoryReportComponent extends BaseComponent {

  render() {
    return html`
      <div @ruleSelected=${this._ruleSelected}>
        <slot></slot>
      </div>
    `;
  }

  private _ruleSelected() {

  }
}
