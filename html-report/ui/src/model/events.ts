export const CategoryActivated = 'categoryActivated';
export const RuleSelected = 'ruleSelected';
export const ViolationSelected = 'violationSelected';

export type RuleSelectedEvent = VacuumEvent;

export interface VacuumEvent {
  id: string;
}

export interface CategoryActivatedEvent extends VacuumEvent {
  description: string;
}

export interface ViolationSelectedEvent extends VacuumEvent {
  message: string;
  startLine: number;
  startCol: number;
  endLine: number;
  endCol: number;
  path: string;
  category: string;
  violationId?: string;
  howToFix?: string;
  renderedCode: Element;
}
