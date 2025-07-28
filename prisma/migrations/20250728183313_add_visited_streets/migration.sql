/*
  Warnings:

  - Added the required column `userName` to the `user_friends` table without a default value. This is not possible if the table is not empty.
  - Made the column `imageUrl` on table `users` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "user_friends" ADD COLUMN     "firstName" TEXT,
ADD COLUMN     "imageUrl" TEXT,
ADD COLUMN     "lastName" TEXT,
ADD COLUMN     "userName" TEXT NOT NULL;

-- AlterTable
ALTER TABLE "users" ALTER COLUMN "imageUrl" SET NOT NULL,
ALTER COLUMN "imageUrl" SET DEFAULT 'https://48htuluf59.ufs.sh/f/1NvBfFppWcZeWF2WCCi3zDay6IgjQLVNYHEhKiCJ8OeGwTon';

-- CreateTable
CREATE TABLE "visited_streets" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "session_id" TEXT NOT NULL,
    "street_id" TEXT NOT NULL,
    "street_name" TEXT NOT NULL,
    "entry_timestamp" BIGINT NOT NULL,
    "exit_timestamp" BIGINT,
    "duration_seconds" INTEGER,
    "entry_latitude" DECIMAL(10,8) NOT NULL,
    "entry_longitude" DECIMAL(11,8) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "visited_streets_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "visited_streets_user_id_idx" ON "visited_streets"("user_id");

-- CreateIndex
CREATE INDEX "visited_streets_session_id_idx" ON "visited_streets"("session_id");

-- CreateIndex
CREATE INDEX "visited_streets_street_id_idx" ON "visited_streets"("street_id");

-- CreateIndex
CREATE INDEX "visited_streets_entry_timestamp_idx" ON "visited_streets"("entry_timestamp");

-- AddForeignKey
ALTER TABLE "visited_streets" ADD CONSTRAINT "visited_streets_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;
