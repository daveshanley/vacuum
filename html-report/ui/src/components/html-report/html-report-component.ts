import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { BaseCSS } from '../../ts/base.css';
import { queryAssignedElements } from 'lit/decorators.js';

export class HtmlReportComponent extends BaseComponent {
  static get styles() {
    const navCss = css`
      .nav-section {
        margin-top: 40px;
      }
    `;
    return [BaseCSS, navCss];
  }

  @queryAssignedElements({
    slot: 'navigation',
    selector: '.category-description',
  })
  _description!: Array<HTMLElement>;

  get _cakes() {
    const slot = this.shadowRoot.querySelector('slot[name=navigation]');
    return slot.getElementsByTagName('section');
  }

  render() {
    return html`
      <div @categoryActivated=${this._categoryActivatedListener}>
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent) {
    const elements = document.querySelectorAll('category-report');
    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');
    if (description) {
      description.innerHTML = e.detail.desc;
    }
    elements.forEach((element: HTMLElement) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });
  }
}
