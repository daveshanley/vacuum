import { BaseComponent } from '../../ts/base-component';
import { css, html, TemplateResult } from 'lit';
import { property } from 'lit/decorators.js';
import { SyntaxCSS } from '../../model/syntax';

export class ViolationDrawerComponent extends BaseComponent {
  static get styles() {
    const listCss = css`
      pre {
        //max-width: 100vw;
        overflow-x: auto;
      }

      p {
        font-size: var(--sl-font-size-x-small);
      }

      pre {
        font-size: var(--sl-font-size-x-small);
      }

      a {
        font-size: var(--sl-font-size-x-small);
        color: var(--primary-color);
      }
      a:hover {
        background-color: var(--secondary-color);
        cursor: pointer;
        color: var(--invert-font-color);
      }
      h2 {
        margin-top: 0;
        line-height: 2.3rem;
        font-size: 1.4rem;
      }

      .backtick-element {
        background-color: black;
        color: var(--secondary-color);
        border: 1px solid var(--secondary-color-lowalpha);
        border-radius: 5px;
        padding: 2px;
      }
    `;
    return [SyntaxCSS, listCss];
  }

  @property()
  code: Element;

  @property()
  message: string;

  @property()
  path: string;

  @property()
  ruleId: string;

  private _visible: boolean;

  render() {
    if (this._visible) {
      return html`
        <h2>${ViolationDrawerComponent.replaceTicks(this.message)}</h2>
        <p>
          Rule Violated:
          <a href="https://quobix.com/vacuum/rules/${this.ruleId}"
            >${this.ruleId}</a
          >
        </p>
        ${this.code}
      `;
    } else {
      return null;
    }
  }

  public show() {
    this._visible = true;
    this.requestUpdate();
  }

  public hide() {
    this._visible = false;
    this.requestUpdate();
  }

  private static replaceTicks(message: string): TemplateResult[] {
    const rx = /(`[^`]*`)/g;

    const sections = message.split(rx);
    console.log('sections', sections);

    const renders: Array<TemplateResult> = new Array<TemplateResult>();

    sections.forEach((section: string) => {
      if (section.match(rx)) {
        renders.push(html`
          <span class="backtick-element">${section.replace(/`/g, '')}</span>
        `);
      } else {
        console.log('section:', section);
        if (section != '') {
          renders.push(html`${section}`);
        }
      }
    });
    return renders;
  }
}
