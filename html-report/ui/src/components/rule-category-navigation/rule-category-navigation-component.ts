import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';
import { RuleCategoryLinkComponent } from './rule-category-link-component';
import { CategoryActivated, CategoryActivatedEvent } from '../../model/events';

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

  protected firstUpdated() {
    // default
    setTimeout(() => {
      const evt = new CustomEvent<CategoryActivatedEvent>(CategoryActivated, {
        bubbles: true,
        composed: true,
        detail: {
          id: this.default,
          description: 'All the categories, for those who like a party.',
        },
      });

      // act like we just clicked all categories.
      this.dispatchEvent(evt);

      // now trigger our own listener.
      this._categoryActivatedListener(evt);
    });
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    console.log('activated!', this._slottedChildren);
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
