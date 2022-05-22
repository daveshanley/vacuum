import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class HtmlReportComponent extends BaseComponent {
  static get styles() {
    const navCss = css`
      .nav-section {
        margin-top: 40px;
      }
    `;
    return [BaseCSS, navCss];
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
    elements.forEach((element: HTMLElement) => {
      if (element.id == e.detail) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });
  }
}
