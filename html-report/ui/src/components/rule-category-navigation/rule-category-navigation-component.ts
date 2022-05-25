import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';
import { RuleCategoryLinkComponent } from './rule-category-link-component';
import { CategoryActivatedEvent } from '../../model/events';

export class RuleCategoryNavigationComponent extends BaseComponent {
  static get styles() {
    const buttonCss = css`
      ul {
        margin: 0;
        padding: 0;
        list-style: none;
      }

      li {
        padding-left: 0;
      }
    `;

    return [buttonCss];
  }

  @property()
  default: string;

  render() {
    return html`
      <ul @categoryActivated=${this._categoryActivatedListener}>
        <slot></slot>
      </ul>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    for (let x = 0; x < this._slottedChildren.length; x++) {
      const child = this._slottedChildren[x] as RuleCategoryLinkComponent;
      if (child.name != e.detail.id) {
        child.disableCategory();
      } else {
        // if it's not already been set, set it (in case of default).
        if (!child.active) {
          child.enableCategory();
        }
      }
    }
  }
}
