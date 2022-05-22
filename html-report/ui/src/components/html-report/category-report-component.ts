import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class CategoryReportComponent extends BaseComponent {
  static get styles() {
    return [BaseCSS];
  }

  render() {
    return html`
      <div @ruleSelected=${this._ruleSelected}>
        <slot></slot>
      </div>
    `;
  }

  private _ruleSelected() {
    // for (let x = 0; x < this._slottedChildren.length; x++) {
    //     const items = this._slottedChildren[x].getElementsByTagName("li")
    //     for (let y = 0; y < items.length; y++) {
    //       const listItem = items[y];
    //       const ruleCollection = listItem.getElementsByTagName('category-rule')
    //       const rule = ruleCollection[0] as CategoryRuleComponent;
    //       //console
    //     }
    // }
    //this.dispatchEvent(evt);
  }
}
