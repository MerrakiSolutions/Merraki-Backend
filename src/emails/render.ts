import {
  PasswordReset,
  type PasswordResetProps,
} from "./templates/PasswordReset.js";
import { AdminInvite, type AdminInviteProps } from "./templates/AdminInvite.js";
import {
  OrderConfirmation,
  type OrderConfirmationProps,
} from "./templates/OrderConfirmation.js";
import {
  DownloadLinks,
  type DownloadLinksProps,
} from "./templates/DownloadLinks.js";
import {
  NewsletterConfirm,
  type NewsletterConfirmProps,
} from "./templates/NewsletterConfirm.js";
import {
  NewsletterCampaign,
  type NewsletterCampaignProps,
} from "./templates/NewsletterCampaign.js";

export const renderEmail = {
  passwordReset: (props: PasswordResetProps) => PasswordReset(props),
  adminInvite: (props: AdminInviteProps) => AdminInvite(props),
  orderConfirmation: (props: OrderConfirmationProps) =>
    OrderConfirmation(props),
  downloadLinks: (props: DownloadLinksProps) => DownloadLinks(props),
  newsletterConfirm: (props: NewsletterConfirmProps) =>
    NewsletterConfirm(props),
  newsletterCampaign: (props: NewsletterCampaignProps) =>
    NewsletterCampaign(props),
};
