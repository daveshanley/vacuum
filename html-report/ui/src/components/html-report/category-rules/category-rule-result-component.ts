import {BaseComponent} from '../../../ts/base-component';
import {html} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {ViolationSelected, ViolationSelectedEvent,} from '../../../model/events';
import categoryRuleResultStyles from './category-rule-result.styles';

@customElement('category-rule-result')
export class CategoryRuleResultComponent extends BaseComponent {
  static styles = categoryRuleResultStyles;

  @property({ type: String })
  message: string;

  @property({ type: String })
  category: string;

  @property({ type: String })
  ruleId: string;

  @property({ type: Number })
  startLine: number;

  @property({ type: Number })
  startCol: number;

  @property({ type: Number })
  endLine: number;

  @property({ type: Number })
  endCol: number;

  @property({ type: String })
  path: string;

  @property({ type: String })
  howToFix: string;

  @property({ type: Boolean })
  selected: boolean;

  private _renderedCode: Element;

  private _violationId: string;

  connectedCallback() {
    super.connectedCallback();
    this._violationId = Math.random().toString(20).substring(2);
  }

  get violationId(): string {
    return this._violationId;
  }

  render() {
    return html` <nav
        aria-label="Violation Navigation"
        class="violation ${this.selected ? 'selected' : ''}"
        @click=${this._violationClicked}
      >
        <div class="line">${this.startLine}</div>
        <div class="message">${this.path}</div>
      </nav>
      <div class="code-render">
        <slot></slot>
      </div>`;
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
      category: this.category,
      howToFix: this.howToFix,
      violationId: this.violationId,
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
