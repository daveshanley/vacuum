import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { CategoryActivated, CategoryActivatedEvent } from '../../model/events';
import ruleCategoryLinkStyles from './rule-category-link.styles';

@customElement('rule-category-link')
export class RuleCategoryLinkComponent extends BaseComponent {
  static styles = ruleCategoryLinkStyles;

  active: boolean;

  @property({ type: String })
  name: string;

  @property({ type: Boolean })
  default: boolean;

  @property({ type: String })
  description: string;

  disableCategory() {
    this.active = false;
    this.requestUpdate();
  }

  enableCategory() {
    this.active = true;
    this.requestUpdate();
  }

  toggleCategory(fireEvent = true) {
    this.active = !this.active;
    if (fireEvent) {
      const options = {
        detail: { id: this.name, description: this.description },
        bubbles: true,
        composed: true,
      };
      this.dispatchEvent(
        new CustomEvent<CategoryActivatedEvent>(CategoryActivated, options)
      );
    }
    this.requestUpdate();
  }

  render() {
    return html`
      <li>
        <a
          href="#"
          class="${this.active ? 'active' : ''}"
          @click=${this.toggleCategory}
        >
          <slot></slot>
        </a>
      </li>
    `;
  }
}
