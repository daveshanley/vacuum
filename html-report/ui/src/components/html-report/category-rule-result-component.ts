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

      .line {
        text-align: center;
        border-radius: 5px;
        min-width: 35px;
        max-width: 35px;
        background-color: var(--card-bgcolor);
        color: var(--tertiary-color);
        font-size: var(--sl-font-size-xx-small);
      }

      .violation {
        display: flex;

        border-top: 1px solid var(--card-bgcolor);
        border-bottom: 1px solid var(--card-bgcolor);
        font-size: var(--sl-font-size-x-small);
        color: var(--font-color);
      }

      .violation:hover {
        background-color: var(--secondary-color);
        cursor: pointer;
        color: var(--invert-font-color);
      }

      .code-render {
        display: none;
      }

      .message {
        margin-left: 5px;
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

  private _renderedCode: Element;

  render() {
    return html` <div class="violation" @click=${this._violationClicked}>
        <div class="line">${this.startLine}</div>
        <div class="message">${this.message}</div>
      </div>
      <div class="code-render"><slot></slot></div>`;
  }

  private _violationClicked() {
    let renderedCode: Element;
    if (this._renderedCode) {
      renderedCode = this._renderedCode;
    } else {
      renderedCode = this._slottedChildren[0];
      this._renderedCode = renderedCode;
    }

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
