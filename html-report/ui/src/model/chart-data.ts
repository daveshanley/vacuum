export interface ReportStatistics {
  filesizeKb: number;
  filesizeBytes: number;
  specType: string;
  specFormat: string;
  version: string;
  references: number;
  externalDocs: number;
  schemas: number;
  parameters: number;
  links: number;
  paths: number;
  operations: number;
  tags: number;
  examples: number;
  enums: number;
  security: number;
  categoryStatistics: Array<CategoryStatistic>;
}

export interface CategoryStatistic {
  categoryId: string;
  categoryName: string;
  numIssues: number;
  warnings: number;
  errors: number;
  info: number;
  hints: number;
}
