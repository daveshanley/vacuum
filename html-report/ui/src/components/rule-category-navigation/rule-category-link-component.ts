import { BaseComponent } from '../../ts/base-component';
import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { CategoryActivated, CategoryActivatedEvent } from '../../model/events';

export class RuleCategoryLinkComponent extends BaseComponent {
  static get styles() {
    const linkItemCss = css`
      li {
        padding-left: 0;
      }

      .active {
        background-color: var(--primary-color);
        color: var(--invert-font-color);\
        font-weight: bold;
      }
      a {
        color: var(--primary-color);
        text-decoration: none;
      }
      a:hover {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }
    `;
    return [linkItemCss];
  }

  active: boolean;

  @property()
  name: string;

  @property()
  default: boolean;

  @property()
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
