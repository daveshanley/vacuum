import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';
import { BaseCSS } from '../../ts/base.css';
import { RuleCategoryLinkComponent } from './rule-category-link-component';

export class RuleCategoryNavigationComponent extends BaseComponent {
  static get styles() {
    const buttonCss = css`
      ul {
        margin: 0;
        padding: 0;
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
      <ul @categoryActive=${this._categoryActivatedListener}>
        <slot></slot>
      </ul>
    `;
  }

  _checkForDefault() {
    if (!this._currentlySelected) {
      const options = {
        detail: { name: this.default, desc: '' },
      };
      this._categoryActivatedListener(
        new CustomEvent('categoryActive', options)
      );
    }
  }

  _categoryActivatedListener(e: CustomEvent) {
    this._currentlySelected = e.detail;
    for (let x = 0; x < this._slottedChildren.length; x++) {
      const child = this._slottedChildren[x] as RuleCategoryLinkComponent;
      if (child.name != e.detail.name) {
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
      detail: { id: e.detail.name, desc: e.detail.desc },
      bubbles: true,
      composed: true,
    };

    // fire a category changed event up to our html-report component.
    this.dispatchEvent(new CustomEvent('categoryActivated', options));
  }
}
