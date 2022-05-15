import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';
import { BaseCSS } from '../../ts/base.css';
import { RuleCategoryButtonComponent } from './rule-category-button-component';

export class RuleCategoryNavigationComponent extends BaseComponent {
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

  private _currentlySelected: string;

  @property()
  default: string;

  render() {
    setTimeout(() => this._checkForDefault());
    return html`
      <nav
        class="category-buttons"
        @categoryActive=${this._categoryActivatedListener}
      >
        <slot></slot>
      </nav>
    `;
  }

  _checkForDefault() {
    if (!this._currentlySelected) {
      const options = {
        detail: this.default,
      };
      this._categoryActivatedListener(
        new CustomEvent('categoryActive', options)
      );
    }
  }

  _categoryActivatedListener(e: CustomEvent) {
    this._currentlySelected = e.detail;
    for (let x = 0; x < this._slottedChildren.length; x++) {
      const child = this._slottedChildren[x] as RuleCategoryButtonComponent;
      if (child.name != e.detail) {
        child.disableCategory();
      } else {
        // if it's not already been set, set it (in case of default).
        if (!child.active) {
          child.enableCategory();
        }
      }
    }

    // options to pass up to html-report.
    const options = {
      detail: e.detail,
      bubbles: true,
      composed: true,
    };
    // fire a category changed event up to our html-report component.
    this.dispatchEvent(new CustomEvent('categoryActivated', options));
  }
}
