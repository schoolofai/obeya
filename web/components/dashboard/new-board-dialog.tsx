"use client";

import React, { useState } from "react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface NewBoardDialogProps {
  open: boolean;
  onClose: () => void;
  onCreate: (name: string) => void;
}

export function NewBoardDialog({ open, onClose, onCreate }: NewBoardDialogProps) {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | undefined>(undefined);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setError("Board name is required");
      return;
    }
    onCreate(name.trim());
    setName("");
    setError(undefined);
  }

  function handleClose() {
    setName("");
    setError(undefined);
    onClose();
  }

  if (!open) return null;

  return (
    <Modal open={open} onClose={handleClose} title="New Board">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Board Name"
          name="boardName"
          placeholder="e.g. Q2 Roadmap"
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={error}
        />
        <DialogActions onClose={handleClose} />
      </form>
    </Modal>
  );
}

interface DialogActionsProps {
  onClose: () => void;
}

function DialogActions({ onClose }: DialogActionsProps) {
  return (
    <div className="flex justify-end gap-3">
      <Button type="button" variant="secondary" onClick={onClose}>
        Cancel
      </Button>
      <Button type="submit" variant="primary">
        Create
      </Button>
    </div>
  );
}
