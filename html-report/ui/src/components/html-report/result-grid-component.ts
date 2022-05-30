import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { SyntaxCSS } from '../../model/syntax';
import { ViolationSelectedEvent } from '../../model/events';
import { ViolationDrawerComponent } from '../violation-drawer/violation-drawer-component';

export class ResultGridComponent extends BaseComponent {
  static get styles() {
    const listCss = css``;
    return [SyntaxCSS, listCss];
  }

  render() {
    return html`
      <slot
        @violationSelected=${this._violationSelectedListener}
        name="violation"
      ></slot>
      <slot name="details"></slot>
    `;
  }

  _violationSelectedListener(e: CustomEvent<ViolationSelectedEvent>) {
    const slots = this.shadowRoot.querySelectorAll('slot');
    const drawer: ViolationDrawerComponent = slots[1].assignedElements({
      flatten: true,
    })[0] as ViolationDrawerComponent;
    drawer.ruleId = e.detail.id;
    drawer.message = e.detail.message;
    drawer.code = e.detail.renderedCode;
    drawer.show();
  }
}
