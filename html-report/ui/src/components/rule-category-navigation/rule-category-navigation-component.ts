import { html, css } from 'lit';
import { state } from 'lit/decorators.js';
import { Category } from '../../model/rule-category';
import { BaseComponent } from '../../ts/base-component';
import { BaseCSS } from '../../ts/base.css';
import { RuleCategoryButtonComponent } from './rule-category-button-component';

export class RuleCategoryNavigationComponent extends BaseComponent {
  static get styles() {
    const rulesCss = css`
      .terminal-menu ul {
        flex-direction: row;
        place-items: center stretch;
        justify-content: center;
      }
    `;
    return [BaseCSS, rulesCss];
  }

  public setCategories(categories: Array<Category>) {
    this._listItems = categories;
    this.requestUpdate();
  }

  toggleCompleted(item: Category) {
    for (let x = 0; x < this._listItems.length; x++) {
      this._listItems[x].active = false;
    }
    item.active = !item.active;
    this.requestUpdate();
  }

  @state()
  private _listItems: Array<Category> = [];

  render() {
    return html`
      <div class="terminal-nav">
        <nav class="terminal-menu">
          <ul @categoryActive=${this._categoryActivatedListener}>
            <slot></slot>
          </ul>
        </nav>
      </div>
    `;
  }

  get _slottedChildren() {
    const slot = this.shadowRoot.querySelector('slot');
    return slot.assignedElements({ flatten: true });
  }

  _categoryActivatedListener(e: CustomEvent) {
    for (let x = 0; x < this._slottedChildren.length; x++) {
      const child = this._slottedChildren[x] as RuleCategoryButtonComponent;
      if (child.name != e.detail) {
        child.disableCategory();
      }
    }
    this.requestUpdate();
  }
}
