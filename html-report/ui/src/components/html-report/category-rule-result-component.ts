import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { BaseCSS } from '../../ts/base.css';

export class CategoryRuleResultComponent extends BaseComponent {
  static get styles() {
    const listCss = css``;

    return [BaseCSS, listCss];
  }

  render() {
    return html` <ul>
      <slot></slot>
    </ul>`;
  }
}
