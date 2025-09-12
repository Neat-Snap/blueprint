"use client";

import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { useTranslations } from "next-intl";

export default function HelpPage() {
  const t = useTranslations('Help');
  return (
    <div className="mx-auto max-w-5xl space-y-8 p-6 md:p-8">
      <div className="space-y-1">
        <h1 className="text-3xl font-semibold tracking-tight">{t('title')}</h1>
        <p className="text-muted-foreground">{t('subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('faq.title')}</CardTitle>
          </CardHeader>
          <CardContent>
            <Accordion type="single" collapsible className="w-full">
              <AccordionItem value="item-1">
                <AccordionTrigger>{t('faq.q1.q')}</AccordionTrigger>
                <AccordionContent>{t('faq.q1.a')}</AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-2">
                <AccordionTrigger>{t('faq.q2.q')}</AccordionTrigger>
                <AccordionContent>{t('faq.q2.a')}</AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-3">
                <AccordionTrigger>{t('faq.q3.q')}</AccordionTrigger>
                <AccordionContent>{t.rich('faq.q3.a', {
                  strong: (chunks) => <span className="font-medium">{chunks}</span>
                })}
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-4">
                <AccordionTrigger>{t('faq.q4.q')}</AccordionTrigger>
                <AccordionContent>{t('faq.q4.a')}</AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-5">
                <AccordionTrigger>{t('faq.q5.q')}</AccordionTrigger>
                <AccordionContent>{t.rich('faq.q5.a', {
                  strong: (chunks) => <span className="font-medium">{chunks}</span>
                })}
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('quick.title')}</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">
            <ol className="list-decimal space-y-2 pl-5 text-muted-foreground">
              <li>{t('quick.step1')}</li>
              <li>{t.rich('quick.step2', { strong: (c) => <span className="font-medium text-foreground">{c}</span> })}</li>
              <li>{t('quick.step3')}</li>
              <li>{t('quick.step4')}</li>
            </ol>
            <Separator className="my-4" />
            <div className="text-foreground">{t('quick.tip')}</div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('account.title')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 text-sm text-muted-foreground">
            <div>
              <div className="font-medium text-foreground">{t('account.item1.title')}</div>
              {t('account.item1.body')}
            </div>
            <Separator />
            <div>
              <div className="font-medium text-foreground">{t('account.item2.title')}</div>
              {t('account.item2.body')}
            </div>
            <Separator />
            <div>
              <div className="font-medium text-foreground">{t('account.item3.title')}</div>
              {t('account.item3.body')}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('resources.title')}</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            <ul className="list-disc space-y-2 pl-5">
              <li>{t('resources.item1')}</li>
              <li>{t('resources.item2')}</li>
              <li>{t('resources.item3')}</li>
            </ul>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('contact.title')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <p className="text-muted-foreground">{t('contact.lead')}</p>
          <ul className="list-disc pl-6 text-muted-foreground">
            <li>{t.rich('contact.item1', { code: (c) => <span className="font-mono">{c}</span> })}</li>
            <li>{t.rich('contact.item2', { code: (c) => <span className="font-mono">{c}</span> })}</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
