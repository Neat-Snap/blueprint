import { Button } from "@/components/ui/button";

export default function Page() {
  return (
    <main className="p-6 space-y-4">
      <h1 className="text-2xl font-bold">Hello shadcn/ui</h1>
      <div className="flex gap-2">
        <Button>Default</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="destructive">Destructive</Button>
      </div>
    </main>
  );
}
