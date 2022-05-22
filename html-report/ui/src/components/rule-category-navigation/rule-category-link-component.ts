import { BaseComponent } from '../../ts/base-component';
import { html, css } from 'lit';
import { BaseCSS } from '../../ts/base.css';
import { property } from 'lit/decorators.js';

export class RuleCategoryLinkComponent extends BaseComponent {
  static get styles() {
    const linkItemCss = css`
      li {
        padding-left: 0;
      }

      .active {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }
    `;
    return [BaseCSS, linkItemCss];
  }

  active: boolean;

  @property()
  name: string;

  @property()
  default: boolean;

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
        detail: this.name,
        bubbles: true,
        composed: true,
      };
      this.dispatchEvent(new CustomEvent('categoryActive', options));
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
