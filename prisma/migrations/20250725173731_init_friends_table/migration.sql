/*
  Warnings:

  - You are about to drop the `posts` table. If the table is not empty, all the data it contains will be lost.

*/
-- CreateEnum
CREATE TYPE "Language" AS ENUM ('En', 'Es', 'Fr', 'De', 'Zh', 'Ja', 'Ru', 'Bg');

-- CreateEnum
CREATE TYPE "Theme" AS ENUM ('Light', 'Dark', 'Auto');

-- CreateEnum
CREATE TYPE "Status" AS ENUM ('ACTIVE', 'SLEEP', 'OFFLINE', 'BANNED', 'PENDING', 'IDLE', 'INVISIBLE');

-- CreateEnum
CREATE TYPE "Role" AS ENUM ('USER', 'ADMIN', 'MODERATOR');

-- DropForeignKey
ALTER TABLE "posts" DROP CONSTRAINT "posts_authorId_fkey";

-- AlterTable
ALTER TABLE "users" ADD COLUMN     "completedTutorial" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "note" TEXT DEFAULT '',
ADD COLUMN     "phoneNumber" TEXT,
ADD COLUMN     "role" "Role" NOT NULL DEFAULT 'USER',
ADD COLUMN     "status" "Status" NOT NULL DEFAULT 'ACTIVE',
ADD COLUMN     "userName" TEXT;

-- DropTable
DROP TABLE "posts";

-- CreateTable
CREATE TABLE "user_friends" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "friend_id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "user_friends_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "city_stats" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "state" TEXT NOT NULL,
    "country" TEXT NOT NULL,
    "population" INTEGER,
    "area" DOUBLE PRECISION,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "userId" TEXT NOT NULL,
    "totalStreetsWalked" INTEGER NOT NULL DEFAULT 0,
    "totalKilometers" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "cityCoveragePct" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "daysActive" INTEGER NOT NULL DEFAULT 0,
    "longestStreakDays" INTEGER NOT NULL DEFAULT 0,
    "settingsId" TEXT,

    CONSTRAINT "city_stats_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "street_walks" (
    "id" TEXT NOT NULL,
    "cityStatId" TEXT NOT NULL,
    "streetName" TEXT NOT NULL,
    "geoJson" JSONB NOT NULL,
    "distanceKm" DOUBLE PRECISION NOT NULL DEFAULT 0,

    CONSTRAINT "street_walks_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "settings" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "theme" "Theme" NOT NULL DEFAULT 'Light',
    "language" "Language" NOT NULL DEFAULT 'En',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "settings_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "user_friends_user_id_friend_id_key" ON "user_friends"("user_id", "friend_id");

-- CreateIndex
CREATE UNIQUE INDEX "city_stats_userId_key" ON "city_stats"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "settings_userId_key" ON "settings"("userId");

-- AddForeignKey
ALTER TABLE "user_friends" ADD CONSTRAINT "user_friends_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_friends" ADD CONSTRAINT "user_friends_friend_id_fkey" FOREIGN KEY ("friend_id") REFERENCES "users"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "city_stats" ADD CONSTRAINT "city_stats_userId_fkey" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "street_walks" ADD CONSTRAINT "street_walks_cityStatId_fkey" FOREIGN KEY ("cityStatId") REFERENCES "city_stats"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "settings" ADD CONSTRAINT "settings_userId_fkey" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;
