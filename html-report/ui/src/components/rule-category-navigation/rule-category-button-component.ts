import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { BaseCSS } from '../../ts/base.css';
import { property } from 'lit/decorators.js';

export class RuleCategoryButtonComponent extends BaseComponent {
  static get styles() {
    return [BaseCSS];
  }

  active: boolean;

  @property()
  name: string;

  disableCategory() {
    this.active = false;
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
      <button
        class="${this.active ? 'btn btn-primary' : 'btn btn-default btn-ghost'}"
        @click="${this.toggleCategory}"
      >
        <slot></slot>
      </button>
    `;
  }
}
