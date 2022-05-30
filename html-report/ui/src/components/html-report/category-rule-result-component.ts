import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { property } from 'lit/decorators.js';
import { ViolationSelected, ViolationSelectedEvent } from '../../model/events';

export class CategoryRuleResultComponent extends BaseComponent {
  static get styles() {
    const listCss = css`
      ul {
        margin-top: 0;
      }

      .violation a {
        font-size: var(--sl-font-size-x-small);
        color: var(--font-color);
      }
      .violation a:hover {
        background-color: var(--secondary-color);
        cursor: pointer;
        color: var(--invert-font-color);
      }

      .code-render {
        display: none;
      }
    `;
    return [listCss];
  }

  @property()
  message: string;

  @property()
  ruleId: string;

  @property()
  startLine: number;

  @property()
  startCol: number;

  @property()
  endLine: number;

  @property()
  endCol: number;

  @property()
  path: string;

  render() {
    return html` <div>
      <span class="violation">
        <a @click=${this._violationClicked}>${this.message}</a>
      </span>
      <div class="code-render"><slot></slot></div>
    </div>`;
  }

  private _violationClicked() {
    const renderedCode: Element = this._slottedChildren[0];

    const violationDetails: ViolationSelectedEvent = {
      message: this.message,
      id: this.ruleId,
      startLine: this.startLine,
      startCol: this.startCol,
      endLine: this.endLine,
      endCol: this.endCol,
      path: this.path,
      renderedCode: renderedCode,
    };

    const options = {
      detail: violationDetails,
      bubbles: true,
      composed: true,
    };
    this.dispatchEvent(
      new CustomEvent<ViolationSelectedEvent>(ViolationSelected, options)
    );
  }
}
