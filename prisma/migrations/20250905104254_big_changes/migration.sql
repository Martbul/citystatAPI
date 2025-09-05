/*
  Warnings:

  - The values [Light,Dark,Auto] on the enum `Theme` will be removed. If these variants are still used in the database, this will fail.
  - You are about to drop the column `area` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `cityCoveragePct` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `country` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `daysActive` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `name` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `population` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `settingsId` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `state` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `totalKilometers` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the column `totalStreetsWalked` on the `city_stats` table. All the data in the column will be lost.
  - You are about to drop the `street_walks` table. If the table is not empty, all the data it contains will be lost.
  - Added the required column `city` to the `city_stats` table without a default value. This is not possible if the table is not empty.

*/
-- CreateEnum
CREATE TYPE "Level" AS ENUM ('Iron', 'Bronze', 'Silver', 'Gold', 'Dimond', 'Platinum', 'Master');

-- CreateEnum
CREATE TYPE "RoleColors" AS ENUM ('NEXTTONAME', 'INNAME', 'DONTSHOW', 'SYNCPROFILECOLORS');

-- CreateEnum
CREATE TYPE "Motion" AS ENUM ('REDUCEDMOTION', 'SYNCWITHDEVICE', 'DONTPLAYGIFWHENPOSSIBLESHOW', 'PLAYEMOJIES');

-- CreateEnum
CREATE TYPE "StickersAnimation" AS ENUM ('ALWAYS', 'ONINTERACTION', 'NEVER');

-- CreateEnum
CREATE TYPE "MessagesAllowance" AS ENUM ('ALLMSG', 'UNREADMAS', 'HIDE');

-- CreateEnum
CREATE TYPE "TextSize" AS ENUM ('BIG', 'MEDIUM', 'SMALL');

-- AlterEnum
BEGIN;
CREATE TYPE "Theme_new" AS ENUM ('LIGHT', 'DARK', 'SYSTEM');
ALTER TABLE "settings" ALTER COLUMN "theme" DROP DEFAULT;
ALTER TABLE "settings" ALTER COLUMN "theme" TYPE "Theme_new" USING ("theme"::text::"Theme_new");
ALTER TYPE "Theme" RENAME TO "Theme_old";
ALTER TYPE "Theme_new" RENAME TO "Theme";
DROP TYPE "Theme_old";
ALTER TABLE "settings" ALTER COLUMN "theme" SET DEFAULT 'LIGHT';
COMMIT;

-- DropForeignKey
ALTER TABLE "street_walks" DROP CONSTRAINT "street_walks_cityStatId_fkey";

-- AlterTable
ALTER TABLE "city_stats" DROP COLUMN "area",
DROP COLUMN "cityCoveragePct",
DROP COLUMN "country",
DROP COLUMN "daysActive",
DROP COLUMN "name",
DROP COLUMN "population",
DROP COLUMN "settingsId",
DROP COLUMN "state",
DROP COLUMN "totalKilometers",
DROP COLUMN "totalStreetsWalked",
ADD COLUMN     "averageDailyDistance" DOUBLE PRECISION NOT NULL DEFAULT 0,
ADD COLUMN     "averageSessionTime" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "averageSpeed" DOUBLE PRECISION NOT NULL DEFAULT 0,
ADD COLUMN     "city" TEXT NOT NULL,
ADD COLUMN     "cityCoveragePercentage" DOUBLE PRECISION NOT NULL DEFAULT 0,
ADD COLUMN     "curentStreakDays" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "favoriteStreet" TEXT,
ADD COLUMN     "totalDaysActive" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "totalKilometersCovered" DOUBLE PRECISION NOT NULL DEFAULT 0,
ADD COLUMN     "totalSessions" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "totalStreetsCovered" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "totalTimeSpent" INTEGER NOT NULL DEFAULT 0;

-- AlterTable
ALTER TABLE "settings" ADD COLUMN     "allowCityStatDataUsage" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "allowDataAnaliticsAndPerformance" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "allowDataPersonalizationUsage" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "allowInAppRewards" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "enableInAppNotifications" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "enableSoundEffects" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "enableVibration" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "enabledLocationTracking" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "fontStyle" TEXT NOT NULL DEFAULT 'default',
ADD COLUMN     "messagesAllowance" "MessagesAllowance" NOT NULL DEFAULT 'ALLMSG',
ADD COLUMN     "motion" "Motion" NOT NULL DEFAULT 'DONTPLAYGIFWHENPOSSIBLESHOW',
ADD COLUMN     "showRoleColors" "RoleColors" NOT NULL DEFAULT 'NEXTTONAME',
ADD COLUMN     "stickersAnimation" "StickersAnimation" NOT NULL DEFAULT 'ALWAYS',
ADD COLUMN     "textSize" "TextSize" NOT NULL DEFAULT 'MEDIUM',
ADD COLUMN     "zoomLevel" TEXT NOT NULL DEFAULT '100',
ALTER COLUMN "theme" SET DEFAULT 'LIGHT';

-- AlterTable
ALTER TABLE "user_friends" ADD COLUMN     "city" TEXT;

-- AlterTable
ALTER TABLE "users" ADD COLUMN     "aboutMe" TEXT,
ADD COLUMN     "city" TEXT,
ADD COLUMN     "cityAllKilometers" DOUBLE PRECISION,
ADD COLUMN     "cityAllStreetsCount" INTEGER,
ADD COLUMN     "cityBboxEast" DOUBLE PRECISION,
ADD COLUMN     "cityBboxNorth" DOUBLE PRECISION,
ADD COLUMN     "cityBboxSouth" DOUBLE PRECISION,
ADD COLUMN     "cityBboxWest" DOUBLE PRECISION,
ADD COLUMN     "cityCountry" TEXT,
ADD COLUMN     "cityDisplayName" TEXT,
ADD COLUMN     "cityLat" DOUBLE PRECISION,
ADD COLUMN     "cityLng" DOUBLE PRECISION,
ADD COLUMN     "cityName" TEXT,
ADD COLUMN     "cityState" TEXT,
ADD COLUMN     "deleteAccount" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "disableAccount" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "lastActivity" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN     "lastLogin" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN     "rankId" TEXT;

-- DropTable
DROP TABLE "street_walks";

-- CreateTable
CREATE TABLE "ranks" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "points" INTEGER NOT NULL DEFAULT 0,
    "level" "Level" NOT NULL DEFAULT 'Iron',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "ranks_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Device" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "settingsId" TEXT,
    "name" TEXT NOT NULL,
    "location" TEXT NOT NULL,
    "lastLoggedIn" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Device_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "ranks_userId_key" ON "ranks"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "Device_userId_key" ON "Device"("userId");

-- AddForeignKey
ALTER TABLE "ranks" ADD CONSTRAINT "ranks_userId_fkey" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Device" ADD CONSTRAINT "Device_userId_fkey" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Device" ADD CONSTRAINT "Device_settingsId_fkey" FOREIGN KEY ("settingsId") REFERENCES "settings"("id") ON DELETE SET NULL ON UPDATE CASCADE;
