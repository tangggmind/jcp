import * as AppBindings from '../../wailsjs/go/main/App';

export const REVIEW_TYPE_DAILY = 'daily_review';
export const REVIEW_TYPE_SUMMARY = 'summary_review';

export interface ReviewArticle {
  id: string;
  type: string;
  date: string;
  title: string;
  filePath: string;
  templateId: string;
  templateName: string;
  summary: string;
  tags: string[];
  stocks: string[];
  profitLoss: number;
  emotion: string;
  disciplineScore?: number;
  imageCount: number;
  createdAt: number;
  updatedAt: number;
}

export interface ReviewArticleDetail {
  article: ReviewArticle;
  content: string;
  warning?: string;
  error?: string;
}

export interface ReviewListRequest {
  query?: string;
  type?: string;
  startDate?: string;
  endDate?: string;
  tags?: string[];
  stocks?: string[];
  page?: number;
  pageSize?: number;
}

export interface ReviewListResult {
  items: ReviewArticle[];
  total: number;
  error?: string;
}

export interface CreateDailyReviewRequest {
  date: string;
  templateId?: string;
  title?: string;
  stocks?: string[];
  marketSnapshot?: string;
}

export interface SaveReviewArticleRequest {
  id: string;
  title?: string;
  content: string;
}

export interface ReviewTemplate {
  id: string;
  name: string;
  description: string;
  content: string;
  isBuiltin: boolean;
  isDefault: boolean;
  createdAt: number;
  updatedAt: number;
}

export interface SaveReviewTemplateRequest {
  id: string;
  name: string;
  description: string;
  content: string;
  isDefault: boolean;
}

export interface SaveReviewImageRequest {
  articleId: string;
  date: string;
  fileName: string;
  mimeType: string;
  dataBase64: string;
}

export interface DownloadReviewImageRequest {
  articleId: string;
  date: string;
  url: string;
}

export interface SaveReviewImageResult {
  assetId: string;
  filePath: string;
  markdownPath: string;
  markdownText: string;
  mimeType: string;
  size: number;
  error?: string;
}

export interface CompareReviewRequest {
  articleIds: string[];
  startDate?: string;
  endDate?: string;
}

export interface CompareReviewItem {
  articleId: string;
  date: string;
  title: string;
  summary: string;
  tags: string[];
  stocks: string[];
  profitLoss: number;
  emotion: string;
  disciplineScore?: number;
  sections: Record<string, string>;
}

export interface CompareStatItem {
  name: string;
  count: number;
}

export interface CompareReviewResult {
  items: CompareReviewItem[];
  tagStats: CompareStatItem[];
  stockStats: CompareStatItem[];
  error?: string;
}

export interface ReviewScreenCaptureResult {
  dataBase64: string;
  width: number;
  height: number;
  error?: string;
}

export interface ReviewOCRRequest {
  dataBase64: string;
  mimeType: string;
}

export interface ReviewOCRResult {
  text: string;
  error?: string;
}

type WailsFunction = (...args: unknown[]) => Promise<unknown>;

function getBinding(name: string): WailsFunction {
  const bindings = AppBindings as unknown as Record<string, WailsFunction | undefined>;
  const fn = bindings[name];
  if (!fn) {
    throw new Error(`Wails 绑定缺失：${name}`);
  }
  return fn;
}

function normalizeArticle(article: ReviewArticle): ReviewArticle {
  return {
    ...article,
    tags: article.tags ?? [],
    stocks: article.stocks ?? [],
  };
}

function normalizeDetail(detail: ReviewArticleDetail): ReviewArticleDetail {
  if (detail.error) {
    throw new Error(detail.error);
  }
  return {
    ...detail,
    article: normalizeArticle(detail.article),
  };
}

async function expectSuccess(result: unknown): Promise<void> {
  if (result !== 'success') {
    throw new Error(String(result || '操作失败'));
  }
}

export const reviewService = {
  async listArticles(req: ReviewListRequest = {}): Promise<ReviewListResult> {
    const result = await getBinding('GetReviewArticles')(req) as ReviewListResult;
    if (result.error) {
      throw new Error(result.error);
    }
    return {
      ...result,
      items: (result.items ?? []).map(normalizeArticle),
    };
  },

  async getArticle(id: string): Promise<ReviewArticleDetail> {
    return normalizeDetail(await getBinding('GetReviewArticle')(id) as ReviewArticleDetail);
  },

  async createDailyReview(req: CreateDailyReviewRequest): Promise<ReviewArticleDetail> {
    return normalizeDetail(await getBinding('CreateDailyReview')(req) as ReviewArticleDetail);
  },

  async saveArticle(req: SaveReviewArticleRequest): Promise<ReviewArticleDetail> {
    return normalizeDetail(await getBinding('SaveReviewArticle')(req) as ReviewArticleDetail);
  },

  async deleteArticle(id: string): Promise<void> {
    await expectSuccess(await getBinding('DeleteReviewArticle')(id));
  },

  async getSummaryArticle(): Promise<ReviewArticleDetail> {
    return normalizeDetail(await getBinding('GetReviewSummaryArticle')() as ReviewArticleDetail);
  },

  async rebuildIndex(): Promise<void> {
    await expectSuccess(await getBinding('RebuildReviewIndex')());
  },

  async listTemplates(): Promise<ReviewTemplate[]> {
    return await getBinding('GetReviewTemplates')() as ReviewTemplate[];
  },

  async saveTemplate(req: SaveReviewTemplateRequest): Promise<ReviewTemplate> {
    return await getBinding('SaveReviewTemplate')(req) as ReviewTemplate;
  },

  async deleteTemplate(id: string): Promise<void> {
    await expectSuccess(await getBinding('DeleteReviewTemplate')(id));
  },

  async savePastedImage(req: SaveReviewImageRequest): Promise<SaveReviewImageResult> {
    const result = await getBinding('SaveReviewPastedImage')(req) as SaveReviewImageResult;
    if (result.error) {
      throw new Error(result.error);
    }
    return result;
  },

  async downloadImage(req: DownloadReviewImageRequest): Promise<SaveReviewImageResult> {
    const result = await getBinding('DownloadReviewImage')(req) as SaveReviewImageResult;
    if (result.error) {
      throw new Error(result.error);
    }
    return result;
  },

  async compareArticles(req: CompareReviewRequest): Promise<CompareReviewResult> {
    const result = await getBinding('CompareReviewArticles')(req) as CompareReviewResult;
    if (result.error) {
      throw new Error(result.error);
    }
    return result;
  },

  async getAssetBase64(filePath: string): Promise<string> {
    const result = await getBinding('GetReviewAssetBase64')(filePath) as string;
    if (!result.startsWith('data:image/')) {
      throw new Error(result);
    }
    return result;
  },

  async captureScreen(): Promise<ReviewScreenCaptureResult> {
    const result = await getBinding('CaptureReviewScreen')() as ReviewScreenCaptureResult;
    if (result.error) {
      throw new Error(result.error);
    }
    if (!result.dataBase64?.startsWith('data:image/')) {
      throw new Error('截图结果无效');
    }
    return result;
  },

  async captureScreenClip(): Promise<ReviewScreenCaptureResult> {
    const result = await getBinding('CaptureReviewScreenClip')() as ReviewScreenCaptureResult;
    if (result.error) {
      throw new Error(result.error);
    }
    if (!result.dataBase64?.startsWith('data:image/')) {
      throw new Error('截图结果无效');
    }
    return result;
  },

  async ocrImage(req: ReviewOCRRequest): Promise<ReviewOCRResult> {
    const result = await getBinding('OCRReviewImage')(req) as ReviewOCRResult;
    if (result.error) {
      throw new Error(result.error);
    }
    return result;
  },
};
