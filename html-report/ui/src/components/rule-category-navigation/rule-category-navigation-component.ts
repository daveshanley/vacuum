import { html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';
import { RuleCategoryLinkComponent } from './rule-category-link-component';
import { CategoryActivated, CategoryActivatedEvent } from '../../model/events';
import ruleCategoryNavigationStyles from './rule-category-navigation.styles';

@customElement('rule-category-navigation')
export class RuleCategoryNavigationComponent extends BaseComponent {
  static styles = ruleCategoryNavigationStyles;

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
    // trigger default
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
    for (let x = 0; x < this._slottedChildren.length; x++) {
      const child = this._slottedChildren[x] as RuleCategoryLinkComponent;
      if (child.name != e.detail.id) {
        if (child.hasAttribute('disableCategory')) child.disableCategory();
      } else {
        // if it's not already been set, set it (in case of default).
        if (!child.active) {
          child.enableCategory();
        }
      }
    }
  }
}
